module Components.NodeUpdate exposing (view)

import Api.Point as Point exposing (Point)
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import UI.Form as Form
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
    let
        osUpdates =
            Point.getAll o.node.points Point.typeVersionOS |> Point.filterDeleted |> List.sortWith Point.sort
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.update
            , text <|
                Point.getText o.node.points Point.typeDescription ""
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            165

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        checkboxInput =
                            NodeInputs.nodeCheckboxInput opts "0"

                        downloadOS =
                            Point.getText o.node.points Point.typeDownloadOS "0"

                        downloading =
                            downloadOS /= ""

                        osDownloaded =
                            Point.getText o.node.points Point.typeOSDownloaded "0"
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeURI "Update Server" "http://..."
                    , textInput Point.typePrefix "Prefix" ""
                    , checkboxInput Point.typeAutoDownload "Auto download"
                    , checkboxInput Point.typeAutoReboot "Auto reboot/install"
                    , if osDownloaded /= "" then
                        column [ spacing 10 ]
                            [ text <| "OS downloaded, reboot to install: " ++ osDownloaded
                            , Form.buttonRow <|
                                [ Form.button
                                    { label = "Discard"
                                    , color = colors.orange
                                    , onPress = opts.onEditNodePoint [ Point Point.typeDiscardDownload "0" opts.now 1 "" 0 ]
                                    }
                                , Form.button
                                    { label = "Reboot"
                                    , color = colors.red
                                    , onPress = opts.onEditNodePoint [ Point Point.typeReboot "0" opts.now 1 "" 0 ]
                                    }
                                ]
                            ]

                      else if downloading then
                        column [ spacing 10 ]
                            [ text <|
                                "Downloading OS version: "
                                    ++ downloadOS
                            ]

                      else
                        column [] <|
                            [ el [ paddingXY 20 0 ] <| text "OS Updates:"
                            , osUpdatesView opts osUpdates
                            ]
                    ]

                else
                    []
               )


osUpdatesView : NodeInputs.NodeInputOptions msg -> List Point -> Element msg
osUpdatesView opt pts =
    table [ paddingEach { top = 0, bottom = 0, right = 0, left = 70 } ]
        { data = pts
        , columns =
            [ { header = text ""
              , width = fill
              , view = \p -> el [ centerY ] <| text p.text
              }
            , { header = text ""
              , width = fill
              , view =
                    \p ->
                        el [ padding 2 ] <|
                            Form.button
                                { label = "install"
                                , color = colors.blue
                                , onPress = opt.onEditNodePoint [ Point Point.typeDownloadOS "0" opt.now 0 p.text 0 ]
                                }
              }
            ]
        }
