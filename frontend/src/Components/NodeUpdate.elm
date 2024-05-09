module Components.NodeUpdate exposing (view)

import Api.Point as Point exposing (Point)
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import UI.Form as Form
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style exposing (colors)
import UI.ViewIf exposing (viewIf)


view : NodeOptions msg -> Element msg
view o =
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

                        osDownloaded =
                            Point.getText o.node.points Point.typeOSDownloaded "0"

                        error =
                            Point.getText o.node.points Point.typeError "0"

                        versionHW =
                            case Point.get o.node.points Point.typeVersionHW "" of
                                Just point ->
                                    "HW: " ++ point.text

                                Nothing ->
                                    ""

                        versionOS =
                            case Point.get o.node.points Point.typeVersionOS "" of
                                Just point ->
                                    "OS: " ++ point.text

                                Nothing ->
                                    ""

                        versionApp =
                            case Point.get o.node.points Point.typeVersionApp "" of
                                Just point ->
                                    "App: " ++ point.text

                                Nothing ->
                                    ""
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , textInput Point.typeURI "Update server" "http://..."
                    , textInput Point.typePrefix "Prefix" ""
                    , checkboxInput Point.typeAutoDownload "Auto download"
                    , checkboxInput Point.typeAutoReboot "Auto reboot/install"
                    , viewIf (versionHW /= "" || versionOS /= "" || versionApp /= "") <|
                        text
                            ("Current version: "
                                ++ versionHW
                                ++ " "
                                ++ versionOS
                                ++ " "
                                ++ versionApp
                            )
                    , if osDownloaded /= "" then
                        column [ spacing 10 ]
                            [ el [ Font.color Style.colors.blue ] <|
                                text <|
                                    "OS downloaded, reboot to install: "
                                        ++ osDownloaded
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

                      else
                        let
                            downloadOS =
                                Point.getText o.node.points Point.typeDownloadOS "0"

                            downloading =
                                downloadOS /= ""
                        in
                        if downloading then
                            column [ spacing 10 ]
                                [ el [ Font.color Style.colors.blue ] <|
                                    text <|
                                        "Downloading OS version: "
                                            ++ downloadOS
                                ]

                        else
                            let
                                osUpdates =
                                    Point.getAll o.node.points Point.typeOSUpdate |> Point.filterDeleted |> List.sortWith Point.sort
                            in
                            column [] <|
                                [ el [ paddingXY 20 0 ] <| text "OS Updates:"
                                , osUpdatesView opts osUpdates
                                ]
                    , el [ Font.color Style.colors.red ] <| text error
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
