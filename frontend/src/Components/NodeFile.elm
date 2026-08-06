module Components.NodeFile exposing (view)

import Api.Point as Point
import Components.NodeOptions exposing (NodeOptions, oToInputO)
import Element exposing (..)
import Element.Border as Border
import Element.Font as Font
import Time
import UI.Form as Form
import UI.Icon as Icon
import UI.NodeInputs as NodeInputs
import UI.Style as Style exposing (colors)
import Utils.Iso8601 exposing (toDateTimeString)


{-| viewProvisioning shows what provisioning did with this file, and only when
provisioning has used it, so a file node holding a CAN database or a serial
configuration is unaffected.
-}
viewProvisioning : String -> String -> String -> List (Element msg)
viewProvisioning hash applied err =
    if hash == "" then
        []

    else
        [ text " "
        , text <| "     Provisioned: " ++ applied
        , text <| "     Applied contents: " ++ String.left 12 hash
        , if err == "" then
            none

          else
            el [ Font.color colors.red ] <| text <| "     Error: " ++ err
        ]


{-| formatCreated reads the point provisioning orders uploads by. A file node
that predates the point has none, which is worth saying rather than showing an
epoch date.
-}
formatCreated : Time.Zone -> Float -> String
formatCreated zone created =
    if created <= 0 then
        "unknown"

    else
        toDateTimeString zone (Time.millisToPosix (round (created * 1000)))


view : NodeOptions msg -> Element msg
view o =
    let
        desc =
            Point.getText o.node.points Point.typeDescription ""

        name =
            Point.getText o.node.points Point.typeName ""
    in
    column
        [ width fill
        , Border.widthEach { top = 2, bottom = 0, left = 0, right = 0 }
        , Border.color Style.colors.black
        , spacing 6
        ]
    <|
        wrappedRow [ spacing 10 ]
            [ Icon.file
            , text <|
                desc
                    ++ " ("
                    ++ name
                    ++ ")"
            ]
            :: (if o.expDetail then
                    let
                        labelWidth =
                            150

                        opts =
                            oToInputO o labelWidth

                        textInput =
                            NodeInputs.nodeTextInput opts "0"

                        checkbox =
                            NodeInputs.nodeCheckboxInput opts "0"

                        binary =
                            Point.getBool o.node.points Point.typeBinary "0"

                        size =
                            Point.getValue o.node.points Point.typeSize "0"

                        data =
                            Point.getText o.node.points Point.typeData "0"

                        hash =
                            Point.getText o.node.points Point.typeHash "0"

                        created =
                            Point.getValue o.node.points Point.typeCreated "0"

                        provisionHash =
                            Point.getText o.node.points Point.typeProvisionHash "0"

                        provisionError =
                            Point.getText o.node.points Point.typeError "0"

                        provisionApplied =
                            case Point.get o.node.points Point.typeProvisionHash "0" of
                                Just p ->
                                    toDateTimeString o.zone p.time

                                Nothing ->
                                    "never"
                    in
                    [ textInput Point.typeDescription "Description" ""
                    , checkbox "binary" "Binary"
                    , text <| " "
                    , text <| "     File name: " ++ name
                    , text <| "     File size: " ++ String.fromFloat size ++ " bytes"
                    , text <| "     File md5: " ++ hash
                    , text <| "     Added: " ++ formatCreated o.zone created
                    , text <| "     Stored data len: " ++ String.fromInt (String.length data) ++ " bytes"
                    , Form.buttonRow
                        [ Form.button
                            { label = "Upload new file"
                            , color = colors.blue
                            , onPress = o.onUploadFile binary
                            }
                        ]
                    , NodeInputs.nodeKeyValueInput opts Point.typeTag "Tags" "Add Tag"
                    ]
                        ++ viewProvisioning provisionHash provisionApplied provisionError

                else
                    []
               )
